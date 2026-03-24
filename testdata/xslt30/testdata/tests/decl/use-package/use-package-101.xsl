<xsl:package 
  name="http://www.w3.org/xslt30tests/use-package-101"
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/use-package-base-002"
  xmlns:q="http://www.w3.org/xslt30tests/use-package-101"
  exclude-result-prefixes="xs p q">
  
  <!-- tests that decimal-formats are local to a package -->
  
  <xsl:use-package name="http://www.w3.org/xslt30tests/use-package-base-002" package-version="1.0.0"/>
  
  
  
    <xsl:decimal-format grouping-separator="^"/>
    <xsl:decimal-format name="a" grouping-separator="$"/>
    
    <xsl:template name="main" visibility="public">
      <out p="{p:format(1234.5)}" q="{q:format(1234.5)}"/>
    </xsl:template>
    
    <xsl:function name="q:format" as="xs:string" visibility="public">
      <xsl:param name="in" as="xs:decimal"/>
      <xsl:value-of select="format-number($in, '0^000.0'), format-number($in, '0$000.0', 'a')"/>
    </xsl:function>
    
  
  
</xsl:package>   