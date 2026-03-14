<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: Test copying of namespace nodes from a literal result element -->

  <xsl:template match="/">
    <doc xmlns:z="http://z.uri/" xmlns="http://default.uri/">
      <chap xmlns:z="http://new-z.uri/" xmlns="http://chap-default.uri/">
        <para xmlns=""/>
      </chap>
    </doc>
  </xsl:template>

</xsl:stylesheet>
