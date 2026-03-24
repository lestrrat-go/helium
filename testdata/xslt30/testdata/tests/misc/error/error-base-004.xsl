<xsl:package 
  name="http://www.w3.org/xslt30tests/error-base-004"
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/error-base-002"
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
    
    
    <xsl:variable name="v-public" as="xs:string" visibility="public" select="'v/public'"/>
    
    <xsl:variable name="v-private" as="xs:string" visibility="private" select="'v/private'"/>
    
    <xsl:variable name="v-final" as="xs:string" visibility="final" select="'v/final'"/>
    
    
    <xsl:template name="t-public" as="xs:string" visibility="public">t-public</xsl:template>
    
    <xsl:template name="t-private" as="xs:string" visibility="private">t-private</xsl:template>
    
    <xsl:template name="t-final" as="xs:string" visibility="final">t-final</xsl:template>
    
    <xsl:template name="t-abstract" as="xs:string" visibility="abstract"/>
    
  
  
</xsl:package>   