<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#axes?>
  <!-- Purpose: Test for completion of tree using all axes. -->

<xsl:template match="/">
  <out>
    <xsl:text>From root: 
</xsl:text>
    <xsl:call-template name="show-four-directions"/>
    <xsl:for-each select="descendant-or-self::*">
       <xsl:text>
------------------------------------------------
</xsl:text>
<xsl:variable name="ename">
       <xsl:text>From-</xsl:text>
       <xsl:value-of select="name()"/>
</xsl:variable>
<xsl:element name="{$ename}"><xsl:text>&#10;</xsl:text>
       <xsl:call-template name="show-four-directions"/>
</xsl:element>
    </xsl:for-each>
  </out>
</xsl:template>

<xsl:template name="show-four-directions">
  <xsl:text>ancestors: </xsl:text>
  <xsl:for-each select="ancestor::*">
    <xsl:value-of select="name()"/><xsl:text> </xsl:text>
  </xsl:for-each><xsl:text>
</xsl:text>
  <xsl:text>preceding: </xsl:text>
  <xsl:for-each select="preceding::*">
    <xsl:value-of select="name()"/><xsl:text> </xsl:text>
  </xsl:for-each><xsl:text>
</xsl:text>
  <xsl:text>self: </xsl:text>
  <xsl:for-each select="self::*">
    <xsl:value-of select="name()"/><xsl:text> </xsl:text>
  </xsl:for-each><xsl:text>
</xsl:text>
  <xsl:text>descendant: </xsl:text>
  <xsl:for-each select="descendant::*">
    <xsl:value-of select="name()"/><xsl:text> </xsl:text>
  </xsl:for-each><xsl:text>
</xsl:text>
  <xsl:text>following: </xsl:text>
  <xsl:for-each select="following::*">
    <xsl:value-of select="name()"/><xsl:text> </xsl:text>
  </xsl:for-each>
</xsl:template>

</xsl:stylesheet>
