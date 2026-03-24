<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- PURPOSE: tests syntax error in a regular expression. -->
  <?spec fo#func-matches?><?error FORX0002?>
  
  <xsl:output method="xml" indent="yes"/>

  <xsl:template match="/">
    <out>;
      <xsl:value-of select="matches(doc, '?', 'm')"/>
    </out>
  </xsl:template>
  
  <xsl:template match="*[local-name()='match']"/>
  
  <xsl:template match="*[local-name()='non-match']">
  <line><xsl:value-of select="."/></line>
  </xsl:template>

</xsl:stylesheet>