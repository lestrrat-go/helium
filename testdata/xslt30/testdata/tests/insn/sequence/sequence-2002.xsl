<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
   <!-- Purpose: xsl:sequence - check reverse() does not modify argument -->

   <xslt:variable name="seq1" as="item()*">
      <xslt:sequence select="(1,2,3)"/>
   </xslt:variable>

   <xslt:template match="/">
      <out>
         <x1><xslt:value-of select="$seq1"/></x1>
         <x2><xslt:value-of select="reverse($seq1)"/></x2>
         <x3><xslt:value-of select="(reverse($seq1),$seq1)"/></x3>
      </out>
   </xslt:template>
</xslt:transform>
