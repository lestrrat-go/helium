<xsl:package 
  name="http://www.w3.org/xslt30tests/override-base-f-002"
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/override-base-f-002"
  exclude-result-prefixes="xs p">
  
 <!-- The function p:children is deliberately simple, and has some simple properties, like
   that the result is always in document order with no duplicates. 
   The idea of this test is to seduce the optimizer into assuming that these properties 
   can be relied on (for example, by eliminating the deduplication step), and then overriding the function with
   a different function that does not have the same properties. -->
  
    <xsl:function name="p:children" as="node()*" visibility="public">
      <xsl:param name="in" as="node()"/>
      <xsl:sequence select="$in/*"/>
    </xsl:function>
 
  <xsl:function name="p:relatives" as="xs:integer" visibility="public">
    <xsl:param name="in" as="node()"/>
    <xsl:sequence select="count(p:children($in)|())"/>
  </xsl:function>
  
</xsl:package>   