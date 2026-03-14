<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://ns.example.com/strip-type"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs my"
                input-type-annotations="strip"
                version="2.0">
<!-- Purpose: Show that when @input-type-annotations="strip", is-nilled property of element  nodes is set to false.-->

   <xslt:template match="/my:userNode">
      <out>
         <xslt:value-of select="nilled(.)"/>
      </out>
   </xslt:template>
</xslt:transform>
