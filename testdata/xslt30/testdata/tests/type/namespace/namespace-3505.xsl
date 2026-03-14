<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
        xmlns:help="http://foobar.com"
        extension-element-prefixes="help">

<?spec xslt#user-defined-top-level?>
    <!-- Purpose: XSLT processor must ignore a top-level element without giving
        and error if it does not recognize the namespace URI. The prefix used
        must still resolve to a URI; but that URI may not be known. -->

<help:Header comment="Header would go here"/>
<help:TOC comment="Table of Contents"/>
<help:template comment="This is the main template" match="doc" process="children"/>

<xsl:template match="doc">
  <out>
    <xsl:value-of select="'Testing '"/>
    <xsl:for-each select="*">
      <xsl:value-of select="."/><xsl:text> </xsl:text>		
    </xsl:for-each>
    <xsl:call-template name="ThatTemp">
      <xsl:with-param name="sam">quos</xsl:with-param>
    </xsl:call-template>
  </out>
</xsl:template>

<help:template comment="Named template" match="*" name="ThatTemp" process="children"/>

<xsl:template name="ThatTemp">
  <xsl:param name="sam">bo</xsl:param>
  <xsl:copy-of select="$sam"/>
</xsl:template>

<help:Footer comment="Footer would go here"/>

</xsl:stylesheet>
