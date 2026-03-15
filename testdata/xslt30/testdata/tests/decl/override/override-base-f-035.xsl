<xsl:package name="http://www.w3.org/xslt30tests/override-base-f-035" package-version="1.0.0"
  version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/override-base-f-035" exclude-result-prefixes="xs p">

  <xsl:variable name="v" select="0"  visibility="public"/>

  <xsl:function name="p:f" as="xs:anyAtomicType" visibility="public">
    <xsl:param name="in" as="xs:integer"/>
    <xsl:sequence select="$in ne $v"/>
  </xsl:function>


</xsl:package>
