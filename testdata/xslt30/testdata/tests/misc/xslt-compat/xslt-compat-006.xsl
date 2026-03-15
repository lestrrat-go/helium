<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test xsl:fallback inside xsl:analyze-string. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/">
      <out>
         <t:analyze-string select="'abracadabra'" regex="b">            
    	       <t:matching-substring>
    		         <t:value-of select="'!'"/>
    	       </t:matching-substring>
    	       <t:non-matching-substring>
    		         <t:value-of select="."/>
    	       </t:non-matching-substring>
    	       <t:fallback>
                 <t:text>Cannot use analyze-string in this version.</t:text>
               </t:fallback>
         </t:analyze-string>
      </out>
   </t:template>
</t:transform>
