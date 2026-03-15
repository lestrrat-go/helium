<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                version="2.0"
                exclude-result-prefixes="xs">
<!-- Purpose: Set a variable under version 1.0, then use it in an XPath on a version 2.0 element to show use of 2.0 XPath rules. -->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:variable name="len"
                  select="substring(/doc/comment(),2,1) + /doc/s"
                  version="1.0"/>

   <xslt:template match="/">
      <out>
         <v>
            <xslt:value-of select="$len"/>
         </v>
         <x>
            <xslt:value-of select="empty(substring(doc/s[@flag='me'],1,$len))"/>
         </x>
      </out>
   </xslt:template>
</xslt:transform>
