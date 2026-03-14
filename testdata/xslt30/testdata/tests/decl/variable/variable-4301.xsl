<?xml version="1.0"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<?spec xslt#variables?>
    <!-- Purpose: test using parameter names with leading underscore -->
 
<xsl:output method="xml" encoding="UTF-8" indent="no"/>

<xsl:template match="/">
  <out>
    <xsl:call-template name="foo">
      <xsl:with-param name="_foo_par" select="/doc/datum/@value"/>
    </xsl:call-template>
  </out>
</xsl:template>

<xsl:template name="foo">
  <xsl:param name="_foo_par"/>

  <xsl:for-each select="$_foo_par">(<xsl:value-of select="position()"/>)</xsl:for-each>
</xsl:template>

</xsl:stylesheet>