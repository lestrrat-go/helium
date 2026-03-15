<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://my.com/"
  exclude-result-prefixes="xs my">

  <!-- PURPOSE: Test QName to string conversion. -->


  <xsl:param name="p" select="'example-uri'"/>
  <!-- force run-time evaluation -->

  <xsl:template match="/">
    <out>
      <xsl:variable name="q" as="xs:QName" select="QName($p, 'ppp:example-lname')"/>
      <t>
        <xsl:value-of select="string($q)"/>
      </t>
      <xsl:variable name="doc">
        <my:element att="3"/>
      </xsl:variable>
      <n>
        <xsl:value-of select="string(node-name($doc/*)), string(node-name($doc/*/@*))"/>
      </n>
      <xsl:variable name="z" select="xs:QName('my:word')"/>
      <w>
        <xsl:value-of select="string($z)"/>
      </w>
    </out>
  </xsl:template>


</xsl:stylesheet>
