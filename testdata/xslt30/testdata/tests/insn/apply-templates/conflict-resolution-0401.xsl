<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:bar="http://bar.com/"
                exclude-result-prefixes="bar"
                version="2.0">

<?spec xslt#conflict?>
<!-- Tests what happens when a "specific" rule and a "general"
     rule have the same priority. Bug reported by Norman Walsh Jan 2002 -->

<xsl:output method="xml"/>

<xsl:template match="bar:foo" priority="5">
  <a>matched bar:foo</a>
</xsl:template>

<xsl:template match="bar:*" priority="5">
  <b>matched bar:*</b>
</xsl:template>

</xsl:stylesheet>
