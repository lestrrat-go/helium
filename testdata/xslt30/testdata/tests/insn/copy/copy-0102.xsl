<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xslt#copy-of?>    
<!-- Test for simple identity transformation, in main template -->                
<xsl:template match="/">
    <out>
      <xsl:copy-of select="OL"/>
    </out>
</xsl:template>

</xsl:stylesheet>