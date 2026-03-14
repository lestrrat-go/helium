<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
 >
<?spec xslt#shallow-copy?>
<!-- Test for simple identity transformation: xsl:copy applied to atomic value  -->                
<xsl:template match="/">
    <out>
        <xsl:copy select="(a/b/c/d/e/f, 'abc')[1]">
           <garbage/>
        </xsl:copy>
    </out>
</xsl:template>


     
</xsl:stylesheet>