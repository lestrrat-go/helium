<xsl:stylesheet
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:bogus="http://bogus">

<!-- Tests "that default namespaces do not apply directly to attributes." -->
  <?spec xpath#node-tests?>

<xsl:template match="*[@up]">
  <foo/>
</xsl:template>

<xsl:template match="*[@bogus:up]">
  <bar/>
</xsl:template>

</xsl:stylesheet>

