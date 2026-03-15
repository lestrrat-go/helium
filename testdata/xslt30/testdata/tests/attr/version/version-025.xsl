<xsl:stylesheet version="25.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<!-- Test supports-namespace-axis property introduced by erratun E14; results of test don't depend on the answer -->
      <?spec xslt#forwards?>


<xsl:template match="/*" use-when="system-property('xsl:supports-namespace-axis')='yes'">
  <out ns="{namespace::xml}"/>  
</xsl:template>

<xsl:template match="/*" use-when="system-property('xsl:supports-namespace-axis')!='yes'">
  <out ns="{namespace-uri-for-prefix(in-scope-prefixes(.)[.='xml'], .)}"/> 
</xsl:template>



</xsl:stylesheet>