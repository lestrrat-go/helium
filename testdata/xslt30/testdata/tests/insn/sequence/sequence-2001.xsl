<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: xsl:sequence - Construction of empty sequence -->

   <xslt:variable name="empty1" as="item()*">
      <xslt:sequence select="()"/>
   </xslt:variable>

   <xslt:variable name="empty2" as="item()*">
      <xslt:sequence select="node/not/found"/>
   </xslt:variable>

   <xslt:template match="/">
      <out>
         <xslt:value-of select="$empty1"/>
         <xslt:value-of select="$empty2"/>
         <xslt:value-of select="empty($empty1)"/>
         <xslt:value-of select="empty($empty2)"/>
      </out>
   </xslt:template>
</xslt:transform>
