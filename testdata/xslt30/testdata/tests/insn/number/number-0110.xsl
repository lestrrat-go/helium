<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    version="3.0">
    
    <!-- count="." is legal in XSLT 3.0 -->
    <!-- Saxon bug 3224 -->
    
    <xsl:variable name="input">
        <root><number>234</number><number>9234</number><number>1234</number></root>
    </xsl:variable>
    
    <xsl:template name="xsl:initial-template">
        <out>
            <xsl:apply-templates select="$input//number"/>
        </out>
    </xsl:template>
    
    <xsl:template match="number">
        <n><xsl:number count="." format="0" level="multiple"/></n>
    </xsl:template>
    
</xsl:stylesheet>