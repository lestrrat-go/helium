<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema">

<!--
It is a static error if a parameter to xsl:iterate is implicitly mandatory.
-->
   <xsl:template name="main">
      <out>
         <xsl:iterate select="1 to 20">
           <xsl:param name="p" as="xs:integer"/>
           <xsl:sequence select="1"/>
         </xsl:iterate>
      </out>
   </xsl:template>
</xsl:stylesheet>
