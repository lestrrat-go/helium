<xsl:package 
  name="http://www.w3.org/xslt30tests/override-base-001"
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/override-base-001"
  exclude-result-prefixes="xs p">
  
  
  
    <xsl:function name="p:f" as="xs:string" visibility="public">
      <xsl:param name="in" as="xs:string"/>
      <xsl:param name="count" as="xs:integer"/>
      <xsl:sequence select="p:f-private($in, $count)"/>
    </xsl:function>
    
    <xsl:function name="p:f-private" as="xs:string" visibility="private">
      <xsl:param name="in" as="xs:string"/>
      <xsl:param name="count" as="xs:integer"/>
      <xsl:sequence select="string-join((1 to $count)!$in, '')"/>
    </xsl:function>
    
    <xsl:function name="p:f-final" as="xs:string" visibility="final">
      <xsl:param name="in" as="xs:string"/>
      <xsl:param name="count" as="xs:integer"/>
      <xsl:sequence select="p:f-private($in, $count)"/>
    </xsl:function>
    
    <xsl:function name="p:f-abstract" as="xs:string" visibility="abstract" >
      <xsl:param name="in" as="xs:string"/>
      <xsl:param name="count" as="xs:integer"/>
    </xsl:function>
    
    <xsl:variable name="p:final-var" select="p:f('+', 3)" visibility="final"/>
    
    <xsl:variable name="p:f-private-ref" select="p:f-private#2" visibility="public"/>
    
    <xsl:variable name="p:f-ref" select="p:f#2" visibility="public"/>
    
    <xsl:variable name="p:f-partial" select="p:f(?, 3)" visibility="public"/>
  
</xsl:package>   