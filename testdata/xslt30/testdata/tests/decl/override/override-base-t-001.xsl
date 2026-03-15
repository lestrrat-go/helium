<xsl:package 
  name="http://www.w3.org/xslt30tests/override-base-101"
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/override-base-101"
  exclude-result-prefixes="xs p">
  
  
  
    <xsl:template name="t" as="xs:string" visibility="public">
      <xsl:param name="in" as="xs:string"/>
      <xsl:param name="count" as="xs:integer"/>
      <xsl:call-template name="t-private">
         <xsl:with-param name="in" select="$in"/>
         <xsl:with-param name="count" select="$count"/>
      </xsl:call-template>
    </xsl:template>
    
    <xsl:template name="t-private" as="xs:string" visibility="private">
      <xsl:param name="in" as="xs:string"/>
      <xsl:param name="count" as="xs:integer"/>
      <xsl:sequence select="string-join((1 to $count)!$in, '')"/>
    </xsl:template>
    
    <xsl:template name="t-final" as="xs:string" visibility="final">
      <xsl:param name="in" as="xs:string"/>
      <xsl:param name="count" as="xs:integer"/>
      <xsl:call-template name="t-private">
         <xsl:with-param name="in" select="$in"/>
         <xsl:with-param name="count" select="$count"/>
      </xsl:call-template>
    </xsl:template>
    
    <xsl:template name="t-abstract" as="xs:string" visibility="abstract" >
      <xsl:param name="in" as="xs:string"/>
      <xsl:param name="count" as="xs:integer"/>
      <xsl:param name="extra" tunnel="yes" as="xs:integer" required="no" select="0"/>
    </xsl:template>
    
    <xsl:template name="u" as="xs:string" visibility="public">
      <xsl:param name="in" as="xs:string"/>
      <xsl:param name="count" as="xs:integer"/>
      <xsl:call-template name="t">
         <xsl:with-param name="in" select="$in"/>
         <xsl:with-param name="count" select="$count"/>
      </xsl:call-template>
    </xsl:template>
    
  
  
</xsl:package>   