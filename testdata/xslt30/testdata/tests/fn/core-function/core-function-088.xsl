<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-substring?>
  <!-- Purpose: Test of 'substring()' function with 'substring()' as an argument.
       The third argument is missing. 
       Uses the fn: namespace explicitly -->

<xsl:template match="doc" xmlns:fn="http://www.w3.org/2005/xpath-functions">
 <out>
   <xsl:value-of select="fn:substring(fn:substring('internalexternalcorrect substring',9),9)"/>
 </out>
</xsl:template>
</xsl:stylesheet>