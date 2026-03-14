<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Test of namespace stuff.  -->
  <?spec fo#func-name?>
  <?spec fo#func-local-name?>
  <?spec fo#func-namespace-uri?>
  <xsl:template match="/*">
    <out>
      <xsl:for-each select="@* | *">
        <xsl:sort select=". instance of element()"/>
        <xsl:sort select="name()"/>
        <r p="0.{position()}" name="{name(.)}" lname="{local-name(.)}" uri="{namespace-uri(.)}"/>
      </xsl:for-each>
      <xsl:apply-templates select="*"/> 
    </out>
  </xsl:template>

  <xsl:template match="*">
    <xsl:for-each select="@* | *">
      <xsl:sort select=". instance of element()"/>
      <xsl:sort select="name()"/>
      <r p="{count(ancestor::*)}.{position()}" name="{name(.)}" lname="{local-name(.)}"
        uri="{namespace-uri(.)}"/>
    </xsl:for-each>
    <xsl:apply-templates select="*"/>
  </xsl:template>

  <xsl:template match="text()"/>


</xsl:stylesheet>
