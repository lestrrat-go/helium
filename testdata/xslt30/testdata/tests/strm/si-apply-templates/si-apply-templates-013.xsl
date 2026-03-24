<xsl:stylesheet version='3.0' 
    xmlns:xsl='http://www.w3.org/1999/XSL/Transform'
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    expand-text="yes">
    
    <xsl:mode streamable='yes' on-no-match='shallow-copy'/>
   
    
    <xsl:template match="/">
        <xsl:variable name="copy">
            <xsl:apply-templates/>
        </xsl:variable>
        <out>{($copy//ITEM)[1] instance of element(*, xs:untyped)}</out>
    </xsl:template>
    
    <xsl:template match='ITEM/AUTHOR'/> 
</xsl:stylesheet>