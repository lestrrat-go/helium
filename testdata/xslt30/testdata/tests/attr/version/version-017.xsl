<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<!-- PURPOSE:  Test xsl:message with select attribute -->
<?spec xslt#message?>

<xsl:template match="doc">
<out>
  <xsl:message select="'message 1: '">A message</xsl:message>
  <xsl:message select="//b">Another message</xsl:message>
  <xsl:message select="'message 3', //b"/>
</out>
</xsl:template>
</xsl:stylesheet>