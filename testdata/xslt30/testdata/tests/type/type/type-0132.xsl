<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:f="http://my.com/" exclude-result-prefixes="f">

  <!-- PURPOSE: Test implicit conversion of untyped attribute to integer,
       in backwards compatibility mode -->
  <?spec xpath#id-function-calls?>     

  <xsl:template match="/">
    <xsl:variable name="x"><e indent="3"/></xsl:variable>
    <out test="{f:string-pad('#', $x/e/@indent)}"/>
  </xsl:template>
  
  <xsl:function name="f:string-pad" as="xs:string">
    <xsl:param name="in" as="xs:string"/>
    <xsl:param name="n" as="xs:integer"/>
    <xsl:sequence select="string-join((for $i in 1 to $n return $in), '')"/>
  </xsl:function>

</xsl:stylesheet>