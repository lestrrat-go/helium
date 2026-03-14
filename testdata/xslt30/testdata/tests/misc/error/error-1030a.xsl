<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                           error if, for any sort key
                           component, the set of sort key
                           values evaluated for all the items in the initial sequence, after any type
                        conversion requested, contains a pair of ordinary values for which the
                        result of the XPath lt operator is an error.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTDE1030?>


  <xsl:template name="main">
      <out>
         <xsl:for-each select="1 to 5, 'fred'">
            <xsl:sort select="."/>
            <xsl:value-of select="."/>
         </xsl:for-each>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
