<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xslt#defining-templates?>
<?error XTTE0505?>
<!-- PURPOSE:   Basic error test of "as" attribute on xsl:template -->

<xsl:output indent="yes"/>

<xsl:template match="doc" as="element()">
  <xsl:variable name="q" select="15"/>
  <zzz><xsl:value-of select="$q" separator=","/></zzz>
  <xsl:sequence select="42"/>
</xsl:template>
 
</xsl:stylesheet>
