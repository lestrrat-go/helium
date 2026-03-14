<?xml version="1.0"?>
<ped:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:ped="http://www.w3.org/1999/XSL/Transform" version="2.0">
<ped:output method="xml"/>
<ped:key name="tk" match="xyz" use="zyx"/>
<xsl:variable name="joe">2</xsl:variable>
<ped:param name="sam" select="' x 4'"/>

<?spec xslt#xslt-namespace?>
    <!-- Purpose: Have more than one prefix bound to the URI of XSLT namespace. -->

<ped:template match="doc">
  <out>
    <ped:value-of select="'Testing '"/>
    <xsl:for-each select="*">
      <ped:value-of select="."/><ped:text> </ped:text>
    </xsl:for-each>

    <ped:text>&#010;</ped:text>
    <xsl:value-of select="$joe"/>
    <ped:value-of select="$sam"/>

    <ped:text>&#010;</ped:text>
    <ped:call-template name="ThatTemp">
      <ped:with-param name="sam">quos</ped:with-param>
    </ped:call-template>

    <xsl:text>&#010;</xsl:text>
    <xsl:call-template name="ThatTemp"/>
  </out>
</ped:template>

<xsl:template name="ThatTemp">
  <xsl:param name="sam">unchanged</xsl:param>
  <ped:copy-of select="$sam"/>
</xsl:template>

</ped:stylesheet>