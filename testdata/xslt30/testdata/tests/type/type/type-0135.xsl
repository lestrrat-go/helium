<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  exclude-result-prefixes="xs">

  <!-- PURPOSE: Test use of xs:anyAtomicType. -->

  <xsl:template match="/">
    <out>
      <q true="{3 instance of xs:anyAtomicType}"/>
      <xsl:variable name="x" as="xs:anyAtomicType" select="4"/>
      <xsl:variable name="y" as="xs:anyAtomicType" select="'foo'"/>
      <q four="{$x}" foo="{$y}"/>
      <q false="{. instance of xs:anyAtomicType}"/>
    </out>
  </xsl:template>



</xsl:stylesheet>
