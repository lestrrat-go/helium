<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:scm="http://ns.saxonica.com/schema-component-model"
    xmlns:cat="http://www.w3.org/2012/10/xslt-test-catalog"
    xmlns:err="http://www.w3.org/2005/xqt-errors"
    exclude-result-prefixes="xs scm cat err"
    expand-text="yes"
    version="3.0">
    
    <!-- Check that the attributes on XSLT elements permitted by the schema all appear in some test stylesheet -->
    
    <!-- Note, the file schema-for-xslt30.scm is generated from the Schema for XSLT 3.0 stylesheets using the Saxon command:
         java net.sf.saxon.Validate schema-for-xslt30.xsd -scmout schema-for-xslt30.scm
    -->
    
    <xsl:param name="scm" select="doc('schema-for-xslt30.scm')"/>
    <xsl:param name="testCatalog" select="doc('../../../catalog.xml')"/>
    
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
    
    <xsl:variable name="permittedElementAttributePairs" as="xs:string*">
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
    
    <xsl:variable name="actualElementAttributePairs" as="xs:string*">
        <xsl:for-each select="$testCatalog/*/cat:test-set/document(@file)//(cat:stylesheet|cat:package)[not(ancestor::cat:test-case//cat:error[not(@code='Q{}UIOP9876')])]/@file">
            <xsl:message>Processing {.}</xsl:message>
            <xsl:try>
                <xsl:variable name="doc" select="document(.)"/>
                <xsl:sequence select="distinct-values(
                    $doc//xsl:*/@Q{}*
                        [not(starts-with(local-name(), '_'))]
                        [not(local-name() = $standardAttributes)]
                        [not(ancestor::*[xs:decimal(@*:version) gt 3.0])]
                        [not(ancestor-or-self::xsl:*[@use-when[not(.='$RUN')]])]
                        /(local-name(..) || '/@' || local-name(.)))"/>    
                <xsl:catch>
                    <xsl:message>{$err:code} {$err:description} - Failed to process {.}</xsl:message>
                </xsl:catch>
            </xsl:try>
        </xsl:for-each>  
    </xsl:variable>
    
    <xsl:template name="showPermittedPairs">
        <xsl:value-of select="$permittedElementAttributePairs"/>
    </xsl:template>
    
    <xsl:template name="showAbsentPairs">
        <xsl:message>Starting</xsl:message>
        <out>
            <!--<permitted>
                <xsl:value-of select="$permittedElementAttributePairs"/>
            </permitted>
            <actual>
                <xsl:value-of select="$actualElementAttributePairs"/>
            </actual>-->
            <absent>
                <xsl:value-of select="$permittedElementAttributePairs[not(. = $actualElementAttributePairs)]" separator="&#xa;"/>
            </absent>
            <spurious>
                <xsl:value-of select="$actualElementAttributePairs[not(. = $permittedElementAttributePairs)]" separator="&#xa;"/>
            </spurious>
        </out>
    </xsl:template>
    
    <xsl:output indent="yes"/>
    
</xsl:stylesheet>