<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
    xmlns:txt="http://www.host.com"
    xmlns:new="http://www.host.com"
    exclude-result-prefixes="txt new">

<?spec xslt#variables?>
  <!-- Purpose: Show that name of variable acts like a real QName. -->
  

<xsl:variable name="txt:me" select="'Wizard'"/>

<xsl:template match="/">
  <out>
    <xsl:value-of select="$new:me"/>
  </out>
</xsl:template>

</xsl:stylesheet>