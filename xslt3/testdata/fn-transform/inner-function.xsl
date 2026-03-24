<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:f="http://example.com/fn">
  <xsl:function name="f:double" as="xs:integer" visibility="public">
    <xsl:param name="n" as="xs:integer"/>
    <xsl:sequence select="$n * 2"/>
  </xsl:function>
</xsl:stylesheet>
