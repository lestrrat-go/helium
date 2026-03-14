<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses an ifexpr in @select of an xsl:sort instruction. (Using string functions in the ifexpr) -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/">
      <out>
         <t:text>Memos by thread:
</t:text>
         <t:for-each select="doc/memo">
            <t:sort select="if (not(starts-with(.,'Re: '))) then         substring(subj,1,string-length(subj)) else         substring(substring-after(subj,'Re: '),1,string-length(substring-after(subj,'Re: ')))"
                    data-type="text"/>
            <t:sort select="time" data-type="text"/>
            <t:value-of select="body"/>
            <t:text>: </t:text>
            <t:value-of select="subj"/>
            <t:text>;
	 </t:text>
         </t:for-each>
      </out>
   </t:template>

   <t:template match="text()"/>
</t:transform>
