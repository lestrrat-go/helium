<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test that result of xsl:supports-backwards-compatibility is correct according to the test dependencies.-->

   <xslt:template match="/">
      <out>
         <xslt:value-of select="system-property('xslt:supports-backwards-compatibility')"/>
      </out>
   </xslt:template>
</xslt:transform>
