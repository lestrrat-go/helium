<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:scm="http://ns.saxonica.com/schema-component-model"
    xmlns:e="http://www.w3.org/1999/XSL/Spec/ElementSyntax"
    xmlns:err="http://www.w3.org/2005/xqt-errors"
    exclude-result-prefixes="xs scm err e"
    version="3.0">
    
    <!-- Check that the attributes on XSLT elements permitted by the schema are consistent with those permitted by the element syntax summaries -->
    
    <!-- Note, the file schema-for-xslt30.scm is generated from the Schema for XSLT 3.0 stylesheets using the Saxon command:
         java net.sf.saxon.Validate schema-for-xslt30.xsd -scmout schema-for-xslt30.scm
    -->
    
    <xsl:param name="scm" select="doc('schema-for-xslt30.scm')"/>
    <xsl:param name="elementCatalog" select="doc('element-catalog.xml')"/>
    
    <xsl:key name="ID" match="*[@id]" use="@id"/>
    
    <xsl:variable name="standardAttributes" as="xs:string*" select="tokenize('
        extension-element-prefixes
        exclude-result-prefixes
        default-mode
        default-validation
        default-collation
        use-when
        xpath-default-namespace
        expand-text
        version
        ', '\s+')"/>
    
    <xsl:variable name="schemaElementAttributePairs" as="xs:string*">
        <xsl:for-each select="$scm//scm:element[@targetNamespace='http://www.w3.org/1999/XSL/Transform']">
            <xsl:variable name="local" select="@name"/>
            <xsl:variable name="type" select="key('ID', @type, $scm)"/>
            <xsl:variable name="attUses" select="$type/scm:attributeUse"/>
            <xsl:variable name="atts" select="$attUses/key('ID', @ref, $scm)"/>
            <xsl:for-each select="$atts[not(starts-with(@name, '_'))][not(@name = $standardAttributes)]">
                <xsl:sequence select="$local || '/@' || @name"/>
            </xsl:for-each>    
        </xsl:for-each>
    </xsl:variable>
    
    <xsl:variable name="syntaxElementAttributePairs" as="xs:string*">
        <xsl:for-each select="$elementCatalog//e:element-syntax/e:attribute[not(@name = $standardAttributes)]">
            <xsl:sequence select="../@name || '/@' || ./@name"/>
        </xsl:for-each>  
    </xsl:variable>
    

    
    <xsl:template name="showDifferences">
        <xsl:message>Starting</xsl:message>
        <out>
            <absentFromSyntax>
                <xsl:value-of select="$schemaElementAttributePairs[not(. = $syntaxElementAttributePairs)]" separator="&#xa;"/>
            </absentFromSyntax>
            <absentFromSchema>
                <xsl:value-of select="$syntaxElementAttributePairs[not(. = $schemaElementAttributePairs)]" separator="&#xa;"/>
            </absentFromSchema>
        </out>
    </xsl:template>
    
    <xsl:output indent="yes"/>
    
</xsl:stylesheet>