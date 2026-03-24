<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#creating-attributes?>
    <!-- Purpose: Create prefixed attribute with namespace requested via attribute. -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:template match="/">
  <out>
    <xsl:attribute name="bee:see" namespace="http://bee.com/">true</xsl:attribute>
  </out>
</xsl:template>

</xsl:stylesheet>