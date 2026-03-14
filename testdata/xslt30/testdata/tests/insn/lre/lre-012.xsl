<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0" >

<?spec xslt#literal-result-element?>
  <!-- Purpose: Try to put out value of a variable as an LRE. Get "$var" literally. -->
  
<xsl:variable name="var" select="'Data'"/>

<xsl:template match="/">
  <out>$var</out>
</xsl:template>

</xsl:stylesheet>