<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xpath-default-namespace="http://www.w3.org/xpath-functions/spec/namespace">

  <!-- test that all new functions listed in the XSLT specification are not implemented (or claim to be available!) in 2.0 -->
  <!-- This test uses The XSLT function catalog file as input. This should be updated from time to time
     with the latest version (modified by removing the DTD reference) -->


  <xsl:template match="/">
    <out>
      <xsl:apply-templates select=".//proto"/>
    </out>
  </xsl:template>

  <xsl:template match="proto">
    <xsl:variable name="lexname" select="@name"/>
    <f name="{$lexname}" arity="{count(arg)}" test="{function-available($lexname, count(arg))}"/>
  </xsl:template>
  
</xsl:stylesheet>
