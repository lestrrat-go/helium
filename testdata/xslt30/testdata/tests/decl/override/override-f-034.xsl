<xsl:package 
  name="http://www.w3.org/xslt30tests/override-034"
  package-version="1.0.1"
  version="3.0"
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:p="http://www.w3.org/xslt30tests/override-base-f-002"
  exclude-result-prefixes="xs p">
  
  <xsl:variable name="data">
    <data>
      <a/>
      <b/>
      <c/>
    </data>
  </xsl:variable>
  
  <xsl:use-package
    name="http://www.w3.org/xslt30tests/override-base-f-002"
    package-version="1.0.0">
  
    <xsl:override>
  
      <xsl:function name="p:children" as="node()*" visibility="public">
        <xsl:param name="in" as="node()"/>
       <xsl:sequence select="$in/*, reverse($in/*)"/>
      </xsl:function>
      
    </xsl:override>  
    
  </xsl:use-package>
  
 
   
    <xsl:template name="xsl:initial-template" visibility="public">
      <out>
        <xsl:sequence select="p:relatives($data/data)"/>
      </out>
    </xsl:template>
    
 
  
</xsl:package>   