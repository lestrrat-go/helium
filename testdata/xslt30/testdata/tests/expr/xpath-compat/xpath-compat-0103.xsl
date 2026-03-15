<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" version="2.0" exclude-result-prefixes="xs">
   <!-- Purpose: Test substring-before() first or second argument being a datatype unknown in 1.0
    but having a string representation, and using BC Mode to force casting. -->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:template match="/" version="1.0">
      <out>
         <first>
            <xslt:value-of select="substring-before(xs:duration('P6Y2M'),2)"/>
         </first>
         <second>
            <xslt:value-of select="substring-before('98:76:14:00:0000',xs:time('14:00:00'))"/>
         </second>
         <third>
            <xslt:value-of
               select="substring-before(xs:hexBinary('7015D3A594'),xs:hexBinary('3a59'))"/>
         </third>
      </out>
   </xslt:template>
</xslt:transform>
