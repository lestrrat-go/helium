<?xml version="1.0"?> 

<?spec xpath#axes?>
  <!-- PURPOSE: Test ancestor and ancestor-or-self axes starting at a
  namespace node. -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
  <xsl:template match="/">
    <out>
       <xsl:for-each select="//namespace::key">
          <xsl:text>
------------------------------------------------
</xsl:text>
          <xsl:text>From </xsl:text>
            <xsl:value-of select="name()"/>
          <xsl:text>: 
</xsl:text>
          <xsl:call-template name="show-four-directions"/>
        </xsl:for-each>
    </out>
  </xsl:template>
  
  <xsl:template name="show-four-directions">
    <xsl:text>ancestor::* </xsl:text>
    <xsl:for-each select="ancestor::*">
      <xsl:value-of select="name()"/><xsl:text> </xsl:text>
    </xsl:for-each><xsl:text>
</xsl:text>
    <xsl:text>ancestor::node(): </xsl:text>
    <xsl:for-each select="ancestor::node()">
      <xsl:value-of select="name()"/><xsl:text> </xsl:text>
    </xsl:for-each><xsl:text>
</xsl:text>
    <xsl:text>ancestor-or-self::* </xsl:text>
    <xsl:for-each select="ancestor-or-self::*">
      <xsl:value-of select="name()"/><xsl:text> </xsl:text>
    </xsl:for-each><xsl:text>
</xsl:text>
    <xsl:text>ancestor-or-self::node() </xsl:text>
    <xsl:for-each select="ancestor-or-self::node()">
      <xsl:value-of select="name()"/><xsl:text> </xsl:text>
    </xsl:for-each><xsl:text>
</xsl:text>
    <xsl:text>following: </xsl:text>
    <xsl:for-each select="following::*">
      <xsl:value-of select="name()"/><xsl:text> </xsl:text>
    </xsl:for-each><xsl:text>
</xsl:text>

    <xsl:text>preceding: </xsl:text>
    <xsl:for-each select="preceding::*">
      <xsl:value-of select="name()"/><xsl:text> </xsl:text>
    </xsl:for-each><xsl:text>
</xsl:text>
  </xsl:template>
 
</xsl:stylesheet>
