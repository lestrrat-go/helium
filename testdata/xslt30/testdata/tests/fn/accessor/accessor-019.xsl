<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Test of fn:data accessor with empty sequence -->

   <xslt:template match="/">
      <out>
         <xslt:value-of select="data(()) instance of empty-sequence()"/>
         <xslt:value-of select="data(doc/inner/*) instance of empty-sequence()"/>
      </out>
   </xslt:template>
</xslt:transform>
