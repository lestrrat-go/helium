<?xml version="1.0" ?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#axes?>
  <!-- Purpose: Comprehensive test of all axes -->
  <!-- Author: Dave Haffner -->

<xsl:strip-space elements="*"/>

<xsl:template match="//a[@id='36']">
<out>
   <xsl:text>matched on node </xsl:text>
   <xsl:value-of select="./@id"/>
   <xsl:text>: </xsl:text>

   <xsl:call-template name="displayNodes">
      <xsl:with-param name="nodeList" select="parent::*" />
      <xsl:with-param name="axisName" select="'parent::*'" />
   </xsl:call-template>

   <xsl:call-template name="displayNodes">
      <xsl:with-param name="nodeList" select="following::*" />
      <xsl:with-param name="axisName" select="'following::*'" />
   </xsl:call-template>

   <xsl:call-template name="displayNodes">
      <xsl:with-param name="nodeList" select="following-sibling::*" />
      <xsl:with-param name="axisName" select="'following-sibling::*'" />
   </xsl:call-template>

   <xsl:call-template name="displayNodes">
      <xsl:with-param name="nodeList" select="preceding::*" />
      <xsl:with-param name="axisName" select="'preceding::*'" />
   </xsl:call-template>

   <xsl:call-template name="displayNodes">
      <xsl:with-param name="nodeList" select="preceding-sibling::*" />
      <xsl:with-param name="axisName" select="'preceding-sibling::*'" />
   </xsl:call-template>

   <xsl:call-template name="displayNodes">
      <xsl:with-param name="nodeList" select="child::*" />
      <xsl:with-param name="axisName" select="'child::*'" />
   </xsl:call-template>

   <xsl:call-template name="displayNodes">
      <xsl:with-param name="nodeList" select="descendant::*" />
      <xsl:with-param name="axisName" select="'descendant::*'" />
   </xsl:call-template>

   <xsl:call-template name="displayNodes">
      <xsl:with-param name="nodeList" select="descendant-or-self::*" />
      <xsl:with-param name="axisName" select="'descendant-or-self::*'" />
   </xsl:call-template>

   <xsl:call-template name="displayNodes">
      <xsl:with-param name="nodeList" select="ancestor::*" />
      <xsl:with-param name="axisName" select="'ancestor::*'" />
   </xsl:call-template>

   <xsl:call-template name="displayNodes">
      <xsl:with-param name="nodeList" select="ancestor-or-self::*" />
      <xsl:with-param name="axisName" select="'ancestor-or-self::*'" />
   </xsl:call-template>

  <!-- Special handling for attribute axis. -->
  <xsl:text>&#10;     Axis: attribute::* (sorted):  </xsl:text>
  <xsl:for-each select="attribute::*" >
    <xsl:sort select="name()"/>
    <xsl:value-of select="name()"/>
    <xsl:text>: </xsl:text>
    <xsl:value-of select="."/>
    <xsl:text>, </xsl:text>
  </xsl:for-each>
</out>
</xsl:template>

<xsl:template name="displayNodes">
   <xsl:param name="nodeList" select="/.."/>
   <xsl:param name="axisName" select="''" />
     Axis: <xsl:value-of select="$axisName"/>
     <xsl:text>:  </xsl:text>
        <xsl:for-each select="$nodeList" >
            <xsl:value-of select="@id"/>
            <xsl:text>,</xsl:text>
        </xsl:for-each>
</xsl:template>

</xsl:stylesheet>
