<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                  error if the unparsed-entity-public-id
                  function is called when there is no context
                     node, or when the root of the tree containing the context node
                  is not a document node.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE1380?>


  <xsl:template name="main">
    
      <out>
        
          <xsl:sequence select="my:f()"/>
        
      </out>
  </xsl:template>
  
  <xsl:function name="my:f">
      <xsl:sequence select="unparsed-entity-public-id('pling')"/>
  </xsl:function>
  



</xsl:stylesheet>
