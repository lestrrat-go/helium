<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                >

<!-- PURPOSE: Test copying of namespace nodes -->
<?spec xslt#copy-of?>
<xsl:output indent="no"/>
<xsl:template match="/">
<out>
  <xsl:copy-of select="."/>
</out>
</xsl:template>
 
</xsl:stylesheet>
