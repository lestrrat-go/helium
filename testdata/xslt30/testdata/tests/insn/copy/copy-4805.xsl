<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
 >
<?spec xslt#shallow-copy?>
<!-- Test for simple identity transformation: xsl:copy applied to sequence of more than one value  -->                
<xsl:template match="/">
    <out>
        <xsl:copy select="//*">
           <garbage/>
        </xsl:copy>
    </out>
</xsl:template>


     
</xsl:stylesheet>