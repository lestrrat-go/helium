<xsl:package 
  name="http://www.w3.org/xslt30tests/override-f-006"
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/override-base-001"
  exclude-result-prefixes="xs p">
  
  <!-- disallow LRE as a child of xsl:override (even in a namespace) -->
  
  <xsl:use-package
    name="http://www.w3.org/xslt30tests/override-base-001"
    package-version="1.0.0">
  
    <xsl:override>
  
      <xsl:function name="p:f" as="xs:string" visibility="public">
        <xsl:param name="in" as="xs:string"/>
        <xsl:param name="count" as="xs:integer"/>
        <xsl:sequence select="'*' || p:f-final($in, $count) || '*'"/>
      </xsl:function>
    
	  <p:out/>
      
    </xsl:override>  
    
  </xsl:use-package>
  
 
   
    <xsl:template name="main" visibility="public">
      <out/>
    </xsl:template>
    
 
  
</xsl:package>   