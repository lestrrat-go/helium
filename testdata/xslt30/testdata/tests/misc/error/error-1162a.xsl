<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">      

              <!--When a URI reference supplied to the document function is a relative reference, 
                  it is a dynamic error 
                  if no base URI is available to resolve the relative reference. This can arise
                  for example when the URI is contained in a node that has no base URI (for example a parentless
                  text node), or when the second argument to the function is a node that has no
                  base URI, or when the base URI from the static context is undefined.
               -->
               
   <xsl:variable name="t" as="text()">
     <xsl:value-of select="'abcd.xml'"/>
   </xsl:variable>
   
   <xsl:template name="main">
      <out>
         <xsl:copy-of select="document($t)"/>
      </out>
   </xsl:template>
</xsl:stylesheet>
