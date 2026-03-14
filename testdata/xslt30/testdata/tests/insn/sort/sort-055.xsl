<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
   <!-- Purpose: Test that collation attribute is accepted for an xsl:sort instruction (use codepoint collation for universality).  -->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:template match="/">
      <out>
         <xslt:for-each select="doc/item">
            <xslt:sort select="@target" lang="en"
               collation="http://www.w3.org/2005/xpath-functions/collation/codepoint"/>
            <e num="{@num}" t="{@target}"/>
         </xslt:for-each>
      </out>
   </xslt:template>
</xslt:transform>
