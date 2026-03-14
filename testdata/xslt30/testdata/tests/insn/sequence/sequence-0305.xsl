<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: range operator - empty sequence in arg2 -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:template match="/">
      <out>
         <xslt:value-of select="6 to subsequence((doc/not/found),3,9)"/>
      </out>
   </xslt:template>
</xslt:transform>
