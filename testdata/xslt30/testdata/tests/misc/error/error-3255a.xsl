<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

              <!--It is a dynamic error if the value of
                  $options includes an entry whose key is indent, 
                  validate, unescape, or fallback,
                  and whose value is not a permitted value for that key.-->
                  
   <xsl:variable name="x" as="element()">
     <null xmlns="http://www.w3.org/2005/xpath-functions"/>
   </xsl:variable>               
                  
   <xsl:template name="main">
      <out>
         <xsl:value-of select="xml-to-json($x, map{'indent':0})"/>
      </out>
   </xsl:template>
</xsl:stylesheet>
