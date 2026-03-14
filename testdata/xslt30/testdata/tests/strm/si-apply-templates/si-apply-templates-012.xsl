<xsl:stylesheet version='3.0' 
    xmlns:xsl='http://www.w3.org/1999/XSL/Transform'
    expand-text="yes">
    
    <xsl:mode streamable='yes' on-no-match='shallow-copy'/>
    
    <xsl:import-schema schema-location="../docs/books.xsd"/>
    
    <xsl:template match="/">
        <xsl:variable name="copy">
            <xsl:apply-templates/>
        </xsl:variable>
        <out>{($copy//ITEM)[1] instance of schema-element(ITEM)}</out>
    </xsl:template>
    
    <xsl:template match='schema-element(ITEM)/AUTHOR'/> 
</xsl:stylesheet>