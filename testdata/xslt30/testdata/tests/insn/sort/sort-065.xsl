<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose:Sort a sequence of comment nodes -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:variable name="list" as="comment()*">
		    <t:comment>First comment</t:comment>
		    <t:comment>XSLT_comment </t:comment>
		    <t:comment>
			      <t:value-of select="'My comment'"/>
		    </t:comment>
		    <t:comment>A comment </t:comment>
		    <t:comment>    </t:comment>
		    <t:comment/>
	  </t:variable>

   <t:template match="/">
	     <out>
         <t:call-template name="temp"/>
      </out>
   </t:template>

   <t:template name="temp">
      <t:perform-sort select="$list">
         <t:sort select="."/>
      </t:perform-sort>
   </t:template>

   <t:template match="text()"/>
</t:transform>
