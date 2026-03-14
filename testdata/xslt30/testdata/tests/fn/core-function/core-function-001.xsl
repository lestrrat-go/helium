<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-substring?>
  <!-- Purpose: Test of 'substring()' function with 'substring()' as an argument.
       The third argument is missing. -->

<xsl:template match="doc">
 <out>
   <xsl:value-of select="substring(substring('internalexternalcorrect substring',9),9)"/>
 </out>
</xsl:template>
</xsl:stylesheet>