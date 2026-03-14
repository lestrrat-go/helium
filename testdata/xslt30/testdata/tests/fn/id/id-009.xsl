<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-id?>
    <!-- Purpose: Test for count() of id(), one value. -->

<xsl:output method="xml" encoding="UTF-8" indent="no"/>

<xsl:template match="/">
  <out>
    <xsl:value-of select="count(id('c'))"/>
  </out>
</xsl:template>

</xsl:stylesheet>