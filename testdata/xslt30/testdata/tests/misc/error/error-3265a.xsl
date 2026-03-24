<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

             <!-- It is a dynamic error if the value of
                  $input is not valid according to the schema for the XML representation of
                  JSON.-->
                  
   <xsl:variable name="x" as="element()">
     <null xmlns="http://www.w3.org/2005/xpath-functions">0</null>
   </xsl:variable>                  
                  
   <xsl:template name="main">
      <out>
         <xsl:value-of select="xml-to-json($x)"/>
      </out>
   </xsl:template>
</xsl:stylesheet>
