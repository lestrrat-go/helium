<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
<?spec fo#func-numeric-divide?>
<!-- PURPOSE:  Test that the result of dividing two integers is a decimal. -->
  <!-- Variant forcing run-time evaluation -->
  
  <xsl:param name="p" select="3"/>

<xsl:template match="/">
<out result="{($p div 4) instance of xs:decimal}"/>
</xsl:template>
</xsl:stylesheet>