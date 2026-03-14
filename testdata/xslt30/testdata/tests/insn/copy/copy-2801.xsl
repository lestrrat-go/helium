<?xml version="1.0" ?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<?spec xslt#copy-of?>
    <!-- Purpose: Using copy-of for repeated output of an RTF (HTML output) -->

  <!-- Source Attribution: This test was written by Michael Kay and is taken from
       'XSLT Programmer's Reference' published by Wrox Press Limited in 2000;
       ISBN 1-861003-12-9; copyright Wrox Press Limited 2000; all rights reserved.
       Now updated in the second edition (ISBN 1-861005-06-7), http://www.wrox.com.
       No part of this book may be reproduced, stored in a retrieval system or transmitted
       in any form or by any means - electronic, electrostatic, mechanical, photocopying,
       recording or otherwise - without the prior written permission of the publisher,
       except in the case of brief quotations embodied in critical articles or reviews. -->
  <!-- Origin: copy-of/soccer.xml, copy-of/soccer.xsl, Chapter/Page: 4-185, was MK015 before -->

<xsl:output method="html" indent="no"/>

<xsl:variable name="table-heading">
  <tr>
    <td><b>Date</b></td>
    <td><b>Home Team</b></td>
    <td><b>Away Team</b></td>
    <td><b>Result</b></td>
  </tr>
</xsl:variable>

<xsl:template match="/">
  <html>
    <body>
      <h1>Matches in Group <xsl:value-of select="/*/@group"/></h1>

      <xsl:for-each select="//match">

        <h2><xsl:value-of select="concat(team[1], ' versus ', team[2])"/></h2>

        <table border="1">
          <xsl:copy-of select="$table-heading"/><!-- This is the test! -->
          <tr>
            <td><xsl:value-of select="date"/>&#xa0;</td>
            <td><xsl:value-of select="team[1]"/>&#xa0;</td>
            <td><xsl:value-of select="team[2]"/>&#xa0;</td>
            <td><xsl:value-of select="concat(team[1]/@score, '-', team[2]/@score)"/>&#xa0;</td>
          </tr>
        </table>
      </xsl:for-each>
    </body>
  </html>
</xsl:template>

</xsl:stylesheet>
