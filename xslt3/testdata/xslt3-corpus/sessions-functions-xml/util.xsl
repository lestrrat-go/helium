<?xml version="1.0" encoding="UTF-8"?>
<!-- Included module defining reusable xsl:functions in the u: namespace. -->
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:u="urn:helium:corpus:util"
    exclude-result-prefixes="xs u">

  <!-- Turn a human title into a URL-safe slug. -->
  <xsl:function name="u:slug" as="xs:string">
    <xsl:param name="s" as="xs:string"/>
    <xsl:sequence select="replace(lower-case(normalize-space($s)), '[^a-z0-9]+', '-')"/>
  </xsl:function>

  <!-- Minutes between two HH:MM clock times. -->
  <xsl:function name="u:duration" as="xs:integer">
    <xsl:param name="start" as="xs:string"/>
    <xsl:param name="end" as="xs:string"/>
    <xsl:variable name="s"
        select="xs:integer(substring-before($start, ':')) * 60
                + xs:integer(substring-after($start, ':'))"/>
    <xsl:variable name="e"
        select="xs:integer(substring-before($end, ':')) * 60
                + xs:integer(substring-after($end, ':'))"/>
    <xsl:sequence select="$e - $s"/>
  </xsl:function>

</xsl:stylesheet>
