<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<xsl:output method="html" indent="no"/>

<?spec xslt#copying?>
<!-- BUG: Saxon 5.3.1/002 -->
<!-- DESCRIPTION: Text content may be lost when xsl:copy-of is used to copy from a result tree fragment
    to an output destination defined with method="html" indent="yes". -->
    
<xsl:variable name="stuff">
  <p><font size="2" face="Verdana, Arial, Helvetica, sans-serif" color="#990033"><b>The list heading</b></font></p><font size="1" face="Verdana, Arial, Helvetica, sans-serif" color="#000000">
     <ul>
        <li>Simultaneous multi-host connectivity for IBM mainframe, AS/400, VAX, OpenVMS, Unix, and Data General</li>
        <li>Multilingual install in English, French, German, or Spanish</li>
        <li>Support for server and single-user installation</li>
     </ul>
  </font>
  <p><font size="1" face="Verdana, Arial, Helvetica, sans-serif" color="#000000">Some text in a paragraph</font></p>
</xsl:variable>

<xsl:template match="/page-items">
  <xsl:copy-of select="$stuff"/>
</xsl:template>

</xsl:stylesheet>

