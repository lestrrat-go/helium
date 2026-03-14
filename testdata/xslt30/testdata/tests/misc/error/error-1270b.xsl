<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                  error to call the key function with two
                  arguments if there is no context
                     node, or if the root of the tree containing the context node is
                  not a document node; or to call the function with three arguments if the
                  root of the tree containing the node supplied in the third argument is not a
                  document node.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE1270?>

   <xsl:key name="k" match="*" use="'pqr'"/>


  <xsl:template name="main">
      <xsl:variable name="e" as="element()">
         <e/>
      </xsl:variable>
      <out>
         <xsl:for-each select="$e">
            <xsl:sequence select="key('k', 'abc')"/>
         </xsl:for-each>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
