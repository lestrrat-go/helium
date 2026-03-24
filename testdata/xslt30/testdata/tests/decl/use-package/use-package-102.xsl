<xsl:package 
  name="http://www.w3.org/xslt30tests/use-package-102"
  package-version="1.0.0"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/use-package-base-002"
  xmlns:q="http://www.w3.org/xslt30tests/use-package-102"
  exclude-result-prefixes="xs p q">
  
  <!-- tests that keys are local to a package -->
  
  <xsl:use-package name="http://www.w3.org/xslt30tests/use-package-base-002" package-version="1.0.0"/>
  
 
    <xsl:key name="k" match="row" use="column[2]"/>
    
    <xsl:function name="q:find" as="element(row)?" visibility="public">
      <xsl:param name="data" as="document-node()"/>
      <xsl:param name="search" as="xs:string"/>
      <xsl:sequence select="key('k', $search, $data)"/>
    </xsl:function>
    
    <xsl:template name="main" visibility="public">
      <out p="{p:find($data, 'aaa')/column[3]}" q="{q:find($data, 'aaa')/column[3]}"/>
    </xsl:template>
    
    <xsl:variable name="data">
      <row>
         <column>aaa</column>
         <column>bbb</column>
         <column>one</column>
      </row>
      <row>
         <column>ccc</column>
         <column>ddd</column>
         <column>two</column>
      </row>
      <row>
         <column>eee</column>
         <column>aaa</column>
         <column>three</column>
      </row>
    </xsl:variable>     
    
  
  
</xsl:package>   