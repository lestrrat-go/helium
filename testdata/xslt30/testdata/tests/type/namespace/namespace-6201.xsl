<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:x="http://example.org/x?subject="
  xmlns:y="http://example.org/x?subject">

  <!-- Purpose: Namespace name ending with "=". Saxon-JS bug 4601-->


  
  <xsl:template match="/">
   <out>
    <xsl:variable name="test">
      <xsl:call-template name="get-test-data"/>
    </xsl:variable>
    <xsl:call-template name="dump-test-data">
      <xsl:with-param name="test" select="$test"/>
    </xsl:call-template>
   </out>
  </xsl:template>
  
  <xsl:template name="get-test-data">
    <x:item>x</x:item>
    <y:item>y</y:item>
  </xsl:template>
  
  <xsl:template name="dump-test-data">
    <xsl:param name="test"/>
    <xsl:for-each select="$test/x:item | $test/y:item">
      <xsl:value-of select="."/>
    </xsl:for-each>
  </xsl:template>
</xsl:stylesheet>
