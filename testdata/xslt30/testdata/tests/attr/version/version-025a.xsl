<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<!-- Test supports-namespace-axis property introduced by erratun E14; results of test don't depend on the answer -->
      <?spec xslt#forwards?>

<xsl:template match="/*">
  <out ns="{namespace-uri-for-prefix(in-scope-prefixes(.)[.='xml'], .)}"/> 
</xsl:template>



</xsl:stylesheet>